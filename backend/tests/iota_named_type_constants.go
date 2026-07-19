package main

type renvoIotaMode int

const (
	renvoIotaModeOff renvoIotaMode = iota
	renvoIotaModeOn
	renvoIotaModeHold
)

func renvoIotaNamedScore(mode renvoIotaMode) int {
	if mode == renvoIotaModeHold {
		return 6
	}
	if mode == renvoIotaModeOn {
		return 3
	}
	return 1
}

func appMain(args []string) int {
	if renvoIotaNamedScore(renvoIotaModeHold) != 6 {
		print("RENVO-IOTA-008 named type failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
