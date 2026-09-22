package main

type switchRecord struct{ value int }

var switchLog int

func switchValue() interface{} {
	switchLog = switchLog*10 + 1
	return "right"
}

func switchCase() interface{} {
	switchLog = switchLog*10 + 2
	return "wrong"
}

func switchUncomparable() (panicked bool) {
	defer func() { panicked = recover() != nil }()
	var x interface{} = []int{1}
	switch x {
	case x:
		return false
	}
	return
}

func appMain(args []string) int {
	var x interface{} = int8(1)
	matched := false
	switch x {
	case int(1):
		panic("distinct case type")
	case int8(1):
		matched = true
	}
	if !matched {
		panic("missing case")
	}
	switch switchValue() {
	case switchCase():
		panic("case value")
	case "right":
		if switchLog != 12 {
			panic("evaluation order")
		}
	default:
		panic("string equality")
	}
	var record interface{} = switchRecord{7}
	switch record {
	case switchRecord{8}:
		panic("record mismatch")
	case switchRecord{7}:
	default:
		panic("record equality")
	}
	var empty interface{}
	switch empty {
	case nil:
	default:
		panic("nil equality")
	}
	if !switchUncomparable() {
		panic("missing comparison panic")
	}
	print("PASS\n")
	return 0
}
