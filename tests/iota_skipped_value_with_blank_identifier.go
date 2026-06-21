package main

const (
	_ = iota
	rtgIotaSkipA
	_
	rtgIotaSkipB
)

func appMain(args []string) int {
	if rtgIotaSkipA != 1 || rtgIotaSkipB != 3 {
		print("RTG-IOTA-010 blank identifier failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
