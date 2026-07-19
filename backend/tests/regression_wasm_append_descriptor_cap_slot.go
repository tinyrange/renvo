package main

type capSlotBox struct {
	values []int
}

var capSlotGlobal []int

func fillGlobal() bool {
	capSlotGlobal = make([]int, 0, 2)
	i := 0
	for i < 9 {
		capSlotGlobal = append(capSlotGlobal, i+10)
		i++
	}
	if len(capSlotGlobal) != 9 {
		return false
	}
	if capSlotGlobal[0] != 10 {
		return false
	}
	return capSlotGlobal[8] == 18
}

func fillField() bool {
	var box capSlotBox
	box.values = make([]int, 0, 2)
	i := 0
	for i < 9 {
		box.values = append(box.values, i+30)
		i++
	}
	if len(box.values) != 9 {
		return false
	}
	if box.values[0] != 30 {
		return false
	}
	return box.values[8] == 38
}

func appMain(args []string) int {
	if fillGlobal() {
		if fillField() {
			print("PASS\n")
			return 0
		}
	}
	print("FAIL\n")
	return 1
}
