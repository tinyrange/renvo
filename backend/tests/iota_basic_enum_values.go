package main

const (
	renvoIotaBasicZero = iota
	renvoIotaBasicOne
	renvoIotaBasicTwo
)

func appMain(args []string) int {
	if renvoIotaBasicZero != 0 || renvoIotaBasicOne != 1 || renvoIotaBasicTwo != 2 {
		print("RENVO-IOTA-001 basic enum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
