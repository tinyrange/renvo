package main

const (
	renvoIotaFlagRead = 1 << iota
	renvoIotaFlagWrite
	renvoIotaFlagExec
)

func appMain(args []string) int {
	if renvoIotaFlagRead != 1 || renvoIotaFlagWrite != 2 || renvoIotaFlagExec != 4 {
		print("RENVO-IOTA-004 shifted flags failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
