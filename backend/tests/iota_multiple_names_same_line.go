package main

const (
	renvoIotaMultiSameA, renvoIotaMultiSameB = iota, iota
	renvoIotaMultiSameC, renvoIotaMultiSameD
)

func appMain(args []string) int {
	if renvoIotaMultiSameA != 0 || renvoIotaMultiSameB != 0 {
		print("RENVO-IOTA-011 first pair failed\n")
		return 1
	}
	if renvoIotaMultiSameC != 1 || renvoIotaMultiSameD != 1 {
		print("RENVO-IOTA-011 second pair failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
