package main

const (
	rtgIotaGlobalA = iota + 4
	rtgIotaGlobalB
)

var rtgIotaGlobalValue int = rtgIotaGlobalB * 2

func appMain(args []string) int {
	if rtgIotaGlobalValue != 10 {
		print("RTG-IOTA-013 global initializer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
