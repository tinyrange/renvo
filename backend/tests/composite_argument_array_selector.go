package main

type compositeArgumentKey struct {
	left  int32
	right [2]int16
}

func compositeArgumentLookup(value compositeArgumentKey) (int32, bool) {
	return value.left + int32(value.right[0]) + int32(value.right[1]), true
}

func appMain(args []string) int {
	first := compositeArgumentKey{left: 7, right: [2]int16{11, 13}}
	value, present := compositeArgumentLookup(compositeArgumentKey{left: first.left, right: first.right})
	if !present || value != 31 {
		print("FAIL: composite argument array selector\n")
		return 1
	}
	print("PASS\n")
	return 0
}
