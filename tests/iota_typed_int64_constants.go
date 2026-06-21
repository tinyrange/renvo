package main

const (
	rtgIotaTypedInt64A int64 = iota + 20
	rtgIotaTypedInt64B
	rtgIotaTypedInt64C
)

func rtgIotaInt64Value() int64 {
	return rtgIotaTypedInt64B
}

func appMain(args []string) int {
	if rtgIotaInt64Value() != 21 || rtgIotaTypedInt64C != 22 {
		print("RTG-IOTA-006 typed int64 failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
