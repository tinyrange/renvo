package main

const (
	renvoIotaReuseA = iota * 3
	renvoIotaReuseB
	renvoIotaReuseC = 20 + iota
	renvoIotaReuseD
)

func appMain(args []string) int {
	if renvoIotaReuseA != 0 || renvoIotaReuseB != 3 {
		print("RENVO-IOTA-009 first reused expression failed\n")
		return 1
	}
	if renvoIotaReuseC != 22 || renvoIotaReuseD != 23 {
		print("RENVO-IOTA-009 second reused expression failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
