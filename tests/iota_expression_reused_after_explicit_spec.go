package main

const (
	rtgIotaReuseA = iota * 3
	rtgIotaReuseB
	rtgIotaReuseC = 20 + iota
	rtgIotaReuseD
)

func appMain(args []string) int {
	if rtgIotaReuseA != 0 || rtgIotaReuseB != 3 {
		print("RTG-IOTA-009 first reused expression failed\n")
		return 1
	}
	if rtgIotaReuseC != 22 || rtgIotaReuseD != 23 {
		print("RTG-IOTA-009 second reused expression failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
