package main

const (
	rtgIotaFlagRead = 1 << iota
	rtgIotaFlagWrite
	rtgIotaFlagExec
)

func appMain(args []string) int {
	if rtgIotaFlagRead != 1 || rtgIotaFlagWrite != 2 || rtgIotaFlagExec != 4 {
		print("RTG-IOTA-004 shifted flags failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
