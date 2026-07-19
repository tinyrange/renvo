package main

const (
	renvoIotaGlobalA = iota + 4
	renvoIotaGlobalB
)

var renvoIotaGlobalValue int = renvoIotaGlobalB * 2

func appMain(args []string) int {
	if renvoIotaGlobalValue != 10 {
		print("RENVO-IOTA-013 global initializer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
