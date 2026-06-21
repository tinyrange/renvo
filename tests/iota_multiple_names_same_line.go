package main

const (
	rtgIotaMultiSameA, rtgIotaMultiSameB = iota, iota
	rtgIotaMultiSameC, rtgIotaMultiSameD
)

func appMain(args []string) int {
	if rtgIotaMultiSameA != 0 || rtgIotaMultiSameB != 0 {
		print("RTG-IOTA-011 first pair failed\n")
		return 1
	}
	if rtgIotaMultiSameC != 1 || rtgIotaMultiSameD != 1 {
		print("RTG-IOTA-011 second pair failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
