package main

var renvo0363Touched int

func renvo0363Mutate() bool { renvo0363Touched = 1; return true }
func appMain(args []string) int {
	if false && renvo0363Mutate() {
		print("RENVO-0363 impossible\n")
		return 1
	}
	if renvo0363Touched != 0 {
		print("RENVO-0363 short and failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
