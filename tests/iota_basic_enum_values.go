package main

const (
	rtgIotaBasicZero = iota
	rtgIotaBasicOne
	rtgIotaBasicTwo
)

func appMain(args []string) int {
	if rtgIotaBasicZero != 0 || rtgIotaBasicOne != 1 || rtgIotaBasicTwo != 2 {
		print("RTG-IOTA-001 basic enum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
