package main

const (
	renvoIotaTypedInt64A int64 = iota + 20
	renvoIotaTypedInt64B
	renvoIotaTypedInt64C
)

func renvoIotaInt64Value() int64 {
	return renvoIotaTypedInt64B
}

func appMain(args []string) int {
	if renvoIotaInt64Value() != 21 || renvoIotaTypedInt64C != 22 {
		print("RENVO-IOTA-006 typed int64 failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
