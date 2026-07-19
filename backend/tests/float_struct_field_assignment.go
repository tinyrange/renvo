package main

type floatFieldAssignmentValue struct {
	value float64
}

func setFloatFieldAssignmentValue(v *floatFieldAssignmentValue) {
	v.value = 1
}

func appMain(args []string) int {
	v := &floatFieldAssignmentValue{}
	setFloatFieldAssignmentValue(v)
	if int(v.value) != 1 {
		print("FAIL integer conversion\n")
		return 1
	}
	if v.value != 1.0 {
		print("FAIL float comparison\n")
		return 1
	}
	print("PASS\n")
	return 0
}
