package main

const (
	renvoIotaPairA, renvoIotaPairB = iota, iota + 10
	renvoIotaPairC, renvoIotaPairD
)

func appMain(args []string) int {
	if renvoIotaPairA != 0 || renvoIotaPairB != 10 {
		print("RENVO-IOTA-012 first values failed\n")
		return 1
	}
	if renvoIotaPairC != 1 || renvoIotaPairD != 11 {
		print("RENVO-IOTA-012 reused values failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
