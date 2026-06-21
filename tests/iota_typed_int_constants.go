package main

const (
	rtgIotaTypedIntA int = iota + 3
	rtgIotaTypedIntB
	rtgIotaTypedIntC
)

func rtgIotaTypedIntScore(x int) int {
	return x * 2
}

func appMain(args []string) int {
	if rtgIotaTypedIntScore(rtgIotaTypedIntC) != 10 {
		print("RTG-IOTA-005 typed int failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
