package main

const (
	_ = iota
	renvoIotaSkipA
	_
	renvoIotaSkipB
)

func appMain(args []string) int {
	if renvoIotaSkipA != 1 || renvoIotaSkipB != 3 {
		print("RENVO-IOTA-010 blank identifier failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
