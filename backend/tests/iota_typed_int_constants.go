package main

const (
	renvoIotaTypedIntA int = iota + 3
	renvoIotaTypedIntB
	renvoIotaTypedIntC
)

func renvoIotaTypedIntScore(x int) int {
	return x * 2
}

func appMain(args []string) int {
	if renvoIotaTypedIntScore(renvoIotaTypedIntC) != 10 {
		print("RENVO-IOTA-005 typed int failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
