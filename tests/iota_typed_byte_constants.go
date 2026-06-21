package main

const (
	rtgIotaTypedByteA byte = iota + 65
	rtgIotaTypedByteB
	rtgIotaTypedByteC
)

func appMain(args []string) int {
	buf := []byte{rtgIotaTypedByteA, rtgIotaTypedByteB, rtgIotaTypedByteC}
	if buf[0] != 'A' || buf[1] != 'B' || buf[2] != 'C' {
		print("RTG-IOTA-007 typed byte failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
