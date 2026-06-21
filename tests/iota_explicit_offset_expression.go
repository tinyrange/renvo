package main

const rtgIotaOffsetBase = 7

const (
	rtgIotaOffsetA = rtgIotaOffsetBase + iota
	rtgIotaOffsetB
	rtgIotaOffsetC
)

func appMain(args []string) int {
	if rtgIotaOffsetA != 7 || rtgIotaOffsetB != 8 || rtgIotaOffsetC != 9 {
		print("RTG-IOTA-003 offset expression failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
