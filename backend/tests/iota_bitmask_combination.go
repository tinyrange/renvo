package main

const (
	renvoIotaMaskRead = 1 << iota
	renvoIotaMaskWrite
	renvoIotaMaskExec
)

func appMain(args []string) int {
	mask := renvoIotaMaskRead | renvoIotaMaskExec
	if mask != 5 {
		print("RENVO-IOTA-016 bitmask value failed\n")
		return 1
	}
	if mask&renvoIotaMaskRead == 0 || mask&renvoIotaMaskWrite != 0 {
		print("RENVO-IOTA-016 bitmask membership failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
