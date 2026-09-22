package main

func appMain(args []string) int {
	const a = 2.0 << 1
	const b = 1 << 2.0
	const c = 20e-1 << 2
	const d = 1 << 2.0000000000000000000000000000000000000000
	const e = -2.0 >> 1
	const f = 1<<2 + 0.5
	if a != 4 || b != 4 || c != 8 || d != 4 || e != -1 || f != 4.5 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
