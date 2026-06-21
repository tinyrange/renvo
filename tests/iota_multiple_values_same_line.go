package main

const (
	rtgIotaPairA, rtgIotaPairB = iota, iota + 10
	rtgIotaPairC, rtgIotaPairD
)

func appMain(args []string) int {
	if rtgIotaPairA != 0 || rtgIotaPairB != 10 {
		print("RTG-IOTA-012 first values failed\n")
		return 1
	}
	if rtgIotaPairC != 1 || rtgIotaPairD != 11 {
		print("RTG-IOTA-012 reused values failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
