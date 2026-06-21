package main

const (
	rtgIotaSliceA = iota + 2
	rtgIotaSliceB
	rtgIotaSliceC
)

func appMain(args []string) int {
	xs := []int{rtgIotaSliceC, rtgIotaSliceA, rtgIotaSliceB}
	if xs[0] != 4 || xs[1] != 2 || xs[2] != 3 {
		print("RTG-IOTA-014 slice literal failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
