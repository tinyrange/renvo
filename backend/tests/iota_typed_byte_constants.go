package main

const (
	renvoIotaTypedByteA byte = iota + 65
	renvoIotaTypedByteB
	renvoIotaTypedByteC
)

func appMain(args []string) int {
	buf := []byte{renvoIotaTypedByteA, renvoIotaTypedByteB, renvoIotaTypedByteC}
	if buf[0] != 'A' || buf[1] != 'B' || buf[2] != 'C' {
		print("RENVO-IOTA-007 typed byte failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
