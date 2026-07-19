package main

const renvoIotaOffsetBase = 7

const (
	renvoIotaOffsetA = renvoIotaOffsetBase + iota
	renvoIotaOffsetB
	renvoIotaOffsetC
)

func appMain(args []string) int {
	if renvoIotaOffsetA != 7 || renvoIotaOffsetB != 8 || renvoIotaOffsetC != 9 {
		print("RENVO-IOTA-003 offset expression failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
