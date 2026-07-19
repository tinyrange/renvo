package main

type Renvo0658IntPtr *int

func appMain(args []string) int {
	value := 8
	var p Renvo0658IntPtr = &value
	for *p < 10 {
		*p = *p + 1
	}
	if value != 10 {
		print("RENVO-0658 named pointer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
