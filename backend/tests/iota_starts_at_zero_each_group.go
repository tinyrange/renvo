package main

const (
	renvoIotaResetA = iota
	renvoIotaResetB
)

const (
	renvoIotaResetC = iota
	renvoIotaResetD
)

func appMain(args []string) int {
	if renvoIotaResetA != 0 || renvoIotaResetB != 1 {
		print("RENVO-IOTA-002 first group failed\n")
		return 1
	}
	if renvoIotaResetC != 0 || renvoIotaResetD != 1 {
		print("RENVO-IOTA-002 second group failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
