package main

type rtgIotaMode int

const (
	rtgIotaModeOff rtgIotaMode = iota
	rtgIotaModeOn
	rtgIotaModeHold
)

func rtgIotaNamedScore(mode rtgIotaMode) int {
	if mode == rtgIotaModeHold {
		return 6
	}
	if mode == rtgIotaModeOn {
		return 3
	}
	return 1
}

func appMain(args []string) int {
	if rtgIotaNamedScore(rtgIotaModeHold) != 6 {
		print("RTG-IOTA-008 named type failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
