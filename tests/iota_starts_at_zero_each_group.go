package main

const (
	rtgIotaResetA = iota
	rtgIotaResetB
)

const (
	rtgIotaResetC = iota
	rtgIotaResetD
)

func appMain(args []string) int {
	if rtgIotaResetA != 0 || rtgIotaResetB != 1 {
		print("RTG-IOTA-002 first group failed\n")
		return 1
	}
	if rtgIotaResetC != 0 || rtgIotaResetD != 1 {
		print("RTG-IOTA-002 second group failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
