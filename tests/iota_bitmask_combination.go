package main

const (
	rtgIotaMaskRead = 1 << iota
	rtgIotaMaskWrite
	rtgIotaMaskExec
)

func appMain(args []string) int {
	mask := rtgIotaMaskRead | rtgIotaMaskExec
	if mask != 5 {
		print("RTG-IOTA-016 bitmask value failed\n")
		return 1
	}
	if mask&rtgIotaMaskRead == 0 || mask&rtgIotaMaskWrite != 0 {
		print("RTG-IOTA-016 bitmask membership failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
