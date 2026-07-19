package main

type pointerAssignParamBox struct {
	value int
}

func pointerAssignParamSet(box *pointerAssignParamBox, value int) {
	if box.value == 0 {
		box.value = value
	}
}

func appMain(args []string) int {
	box := pointerAssignParamBox{}
	pointerAssignParamSet(&box, 19)
	if box.value != 19 {
		print("pointer field assign parameter failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
